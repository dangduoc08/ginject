package devtool

import (
	context "context"
)

type Devtool struct {
	GetConfigurationResponse
	DevtoolServiceServer
}

func (devtool *Devtool) GetConfiguration(context.Context, *GetConfigurationRequest) (*GetConfigurationResponse, error) {

	return &GetConfigurationResponse{
		Controller: devtool.Controller,
	}, nil
}

// Serve is not implemented: the gRPC transport was never wired up, so the
// devtool snapshot is built but never exposed. It blocks until ctx is done so a
// future implementation inherits a shutdown path instead of leaking a goroutine.
func (devtool *Devtool) Serve(ctx context.Context) error {
	<-ctx.Done()

	return ctx.Err()
}
