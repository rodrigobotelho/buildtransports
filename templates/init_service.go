package service

func initService() service.ExampleService {
	return service.New(getServiceMiddleware(logger))
}

