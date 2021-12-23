package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *PostServer) postEvent(ctx *gin.Context) {
	processPostedEvent(ctx, s.Server, s.postEventCallback)
}

func (s *PostServer) postEventCallback(ctx *gin.Context, postLog logr.Logger, podName string, event *Event) error {
	pod := &corev1.Pod{}
	err := s.Client.Get(ctx, client.ObjectKey{Namespace: s.Config.Namespace, Name: podName}, pod)
	if err != nil {
		err = fmt.Errorf("error finding pod: %w", err)
		ctx.String(http.StatusNotFound, err.Error())
		postLog.Error(err, "")
		return err
	}

	if len(event.Args) > 0 {
		s.EventRecorder.Eventf(pod, event.Type(), event.Reason, event.Message, event.args()...)
	} else {
		s.EventRecorder.Event(pod, event.Type(), event.Reason, event.Message)
	}
	return nil
}

type processPostedEventCallback func(ctx *gin.Context, postLog logr.Logger, podName string, event *Event) error

func processPostedEvent(ctx *gin.Context, s *Server, callback processPostedEventCallback) {
	node, executionID := nodeAndID(ctx)
	postLog := s.Log.WithValues(
		"node", node,
		"id", executionID,
	)
	body, err := ctx.GetRawData()
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		postLog.Error(err, "error reading body")
		return
	}
	postLog = postLog.WithValues(
		"length", len(body),
	)

	event := new(Event)
	err = json.NewDecoder(bytes.NewReader(body)).Decode(&event)
	if err != nil {
		ctx.String(http.StatusBadRequest, fmt.Sprintf("error decoding event: %s", err.Error()))
		postLog.WithValues("event", string(body)).Error(err, "error decoding event")
		return
	}

	err = event.Validate()
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		postLog.Error(err, "event is invalid")
		return
	}
	podName := s.Config.PodName(node, executionID)

	if callback(ctx, postLog, podName, event) != nil {
		return
	}

	postLog.WithValues(
		"pod", podName,
		"type", event.Type(),
		"reason", event.Reason,
		"event-message", fmt.Sprintf(event.Message, event.args()...),
	).Info("event created")
}
