package config

import observer "github.com/imkira/go-observer"

type QuitChannel chan string

type ReloadChannel chan ReloadPayload

type ReloadPayload struct {
	Service string
	NewHash string
	OldHash string
}

type ServicePayload struct {
	NodeName   string
	Services   observer.Property
	QuitCh     QuitChannel
	ReloadCh   ReloadChannel
	ListenPort int
}
