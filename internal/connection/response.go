package connection

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

func requestID(req *http.Request) json.RawMessage {
	if req.GetBody == nil {
		return nil
	}
	body, err := req.GetBody()
	if err != nil {
		return nil
	}
	defer body.Close()
	var message struct{ ID json.RawMessage }
	if json.NewDecoder(body).Decode(&message) != nil {
		return nil
	}
	return message.ID
}

// The SDK leaves a POST's SSE call pending if the stream ends without a
// response or a resumable event ID. Detect that end for the particular call so
// a broken response returns an unknown outcome instead of waiting indefinitely.
type responseStream struct {
	io.ReadCloser
	reader   *bufio.Reader
	id       json.RawMessage
	data     bytes.Buffer
	pending  []byte
	finished bool
	onLoss   func()
}

func (s *responseStream) Read(p []byte) (int, error) {
	if len(s.pending) == 0 {
		line, err := s.reader.ReadBytes('\n')
		if err != nil {
			if !s.finished {
				s.onLoss()
			}
			return copy(p, line), err
		}
		field := bytes.TrimRight(line, "\r\n")
		switch {
		case len(field) == 0:
			var message struct {
				ID     json.RawMessage
				Result json.RawMessage
				Error  json.RawMessage
			}
			if json.Unmarshal(s.data.Bytes(), &message) == nil &&
				bytes.Equal(s.id, message.ID) && (message.Result != nil || message.Error != nil) {
				s.finished = true
			}
			s.data.Reset()
		case bytes.HasPrefix(field, []byte("data:")):
			value := field[len("data:"):]
			value = bytes.TrimPrefix(value, []byte(" "))
			s.data.Write(value)
			s.data.WriteByte('\n')
		}
		s.pending = line
	}
	n := copy(p, s.pending)
	s.pending = s.pending[n:]
	return n, nil
}

var errResumeDisabled = errors.New("the CLI does not resume HTTP streams")
