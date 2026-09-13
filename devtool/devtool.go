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

func (devtool *Devtool) Serve() {

}
