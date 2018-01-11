package huobi

type Signer interface {
	GetHuobiKey() string
	HuobiSign(msg string) string
}
