package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/smallbiznis/railzway/internal/usage/liveevents"
)

func (s *Server) StreamMeterLiveEvents(c *gin.Context) {
	if s.liveMeterEvents == nil {
		AbortWithError(c, ErrServiceUnavailable)
		return
	}

	meterID := strings.TrimSpace(c.Param("id"))
	if meterID == "" {
		AbortWithError(c, invalidRequestError())
		return
	}

	meter, err := s.meterSvc.GetByID(c.Request.Context(), meterID)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	subscription, backlog, err := s.liveMeterEvents.Subscribe(meter.Code)
	if err != nil {
		AbortWithError(c, ErrServiceUnavailable)
		return
	}
	defer subscription.Close()

	writer := c.Writer
	headers := writer.Header()
	headers.Set("Content-Type", "text/event-stream")
	headers.Set("Cache-Control", "no-cache")
	headers.Set("Connection", "keep-alive")
	headers.Set("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	flusher, ok := writer.(http.Flusher)
	if !ok {
		AbortWithError(c, ErrServiceUnavailable)
		return
	}

	if _, err := io.WriteString(writer, "retry: 2000\n\n"); err != nil {
		return
	}

	for _, event := range backlog {
		if err := writeLiveMeterEvent(writer, meterID, event); err != nil {
			return
		}
	}
	flusher.Flush()

	ctx := c.Request.Context()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-subscription.Events():
			if err := writeLiveMeterEvent(writer, meterID, event); err != nil {
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			if _, err := io.WriteString(writer, ": heartbeat\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func writeLiveMeterEvent(w io.Writer, meterID string, event liveevents.LiveEvent) error {
	payload := event
	if payload.MeterID == "" {
		payload.MeterID = meterID
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", data)
	return err
}
