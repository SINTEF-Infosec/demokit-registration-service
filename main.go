package main

import "demokit-registration-service/registration"

func main() {
	server := registration.NewRegistrationServer()
	server.Start()
}
