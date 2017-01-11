package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/go-kit/kit/endpoint"
	log "github.com/go-kit/kit/log"
	kithttp "github.com/go-kit/kit/transport/http"
)

// GreetService is the interface that defines our service.
type GreetService interface {
	Hello(string) (string, error)
}

type greetService struct{}

type helloRequest struct {
	Name string `json:"name,omitempty"`
}

type helloResponse struct {
	Greeting string `json:"greeting,omitempty"`
	Err      error  `json:"err,omitempty"`
}

func decodeHelloRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request helloRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func encodeHelloResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	return json.NewEncoder(w).Encode(response)
}

// Hello says hello.
func (g greetService) Hello(s string) (string, error) {
	if s == "" {
		return "", errors.New("no name provided")
	}
	return "Hello there, " + strings.Title(s), nil
}

func makeHelloEndpoint(svc GreetService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(helloRequest)
		resp, err := svc.Hello(req.Name)
		if err != nil {
			return helloResponse{resp, err}, nil
		}
		return helloResponse{resp, nil}, nil
	}
}

type loggingMiddleware struct {
	logger log.Logger
	next   GreetService
}

// Hello logs greetings.
func (mw loggingMiddleware) Hello(s string) (output string, err error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "Hello",
			"input", s,
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	output, err = mw.next.Hello(s)
	return
}

func main() {
	ctx := context.Background()
	logger := log.NewLogfmtLogger(os.Stderr)

	var svc GreetService
	svc = greetService{}
	svc = loggingMiddleware{logger, svc}

	helloHandler := kithttp.NewServer(
		ctx,
		makeHelloEndpoint(svc),
		decodeHelloRequest,
		encodeHelloResponse,
	)

	http.Handle("/hello", helloHandler)
	logger.Log("msg", "HTTP", "addr", ":8080")
	logger.Log("err", http.ListenAndServe(":8080", nil))

}
