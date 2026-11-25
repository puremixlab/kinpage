package services

import (
	"errors"
	"math/big"
	"regexp"
	"strings"
)

var tipEthAddressRe = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

// BuildTipPaymentURI constructs an EIP-681 Ethereum payment URI
// (ethereum:<address>@1?value=<wei>) for WalletConnect / desktop wallet apps
// using exact decimal arithmetic without floating-point rounding.
func BuildTipPaymentURI(recipient, ethAmount string) (string, error) {
	recipient = strings.TrimSpace(recipient)
	if !tipEthAddressRe.MatchString(recipient) && !strings.HasSuffix(strings.ToLower(recipient), ".eth") {
		return "", errors.New("收款地址必须是 0x 以太坊地址或 .eth 域名")
	}
	wei, err := ethDecimalToWei(strings.TrimSpace(ethAmount))
	if err != nil {
		return "", err
	}
	return "ethereum:" + recipient + "@1?value=" + wei, nil
}

func ethDecimalToWei(amount string) (string, error) {
	if amount == "" {
		return "", errors.New("打赏金额不能为空")
	}
	r, ok := new(big.Rat).SetString(amount)
	if !ok || r.Sign() <= 0 {
		return "", errors.New("打赏金额必须是正数")
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	r.Mul(r, new(big.Rat).SetInt(scale))
	wei := new(big.Int).Quo(r.Num(), r.Denom())
	if wei.Sign() <= 0 {
		return "", errors.New("打赏金额过小")
	}
	return wei.String(), nil
}
