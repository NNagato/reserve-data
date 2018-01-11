package huobi

import "os"

type Interface interface {
	PublicEndpoint() string
	AuthenticatedEndpoint() string
}

type RealInterface struct{}

func (self *RealInterface) PublicEndpoint() string {
	return "https://api.huobi.pro"
}

func (self *RealInterface) AuthenticatedEndpoint() string {
	return "https://api.huobi.pro"
}

func NewRealInterface() *RealInterface {
	return &RealInterface{}
}

type SimulatedInterface struct{}

func (self *SimulatedInterface) baseurl() string {
	baseurl := "127.0.0.1"
	if len(os.Args) > 1 {
		baseurl = os.Args[1]
	}
	return baseurl + ":5100"
}

func (self *SimulatedInterface) PublicEndpoint() string {
	return self.baseurl()
}

func (self *SimulatedInterface) AuthenticatedEndpoint() string {
	return self.baseurl()
}

func NewSimulatedInterface() *SimulatedInterface {
	return &SimulatedInterface{}
}

type RopstenInterface struct{}

func (self *RopstenInterface) baseurl() string {
	baseurl := "127.0.0.1"
	if len(os.Args) > 1 {
		baseurl = os.Args[1]
	}
	return baseurl + ":5100"
}

func (self *RopstenInterface) PublicEndpoint() string {
	return "https://api.huobi.pro"
}

func (self *RopstenInterface) AuthenticatedEndpoint() string {
	return self.baseurl()
}

func NewRopstenInterface() *RopstenInterface {
	return &RopstenInterface{}
}

type DevInterface struct{}

func (self *DevInterface) PublicEndpoint() string {
	return "https://api.huobi.pro"
	// return "http://192.168.25.16:5100"
}

func (self *DevInterface) AuthenticatedEndpoint() string {
	return "https://api.huobi.pro"
	// return "http://192.168.25.16:5100"
}

func NewDevInterface() *DevInterface {
	return &DevInterface{}
}
