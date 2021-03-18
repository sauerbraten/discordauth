package auth

import (
	"fmt"

	"github.com/sauerbraten/waiter/pkg/protocol/role"
)

type Provider interface {
	GenerateChallenge(name string, callback func(reqID uint32, chal string, err error))
	ConfirmAnswer(reqID uint32, answ string, callback func(ok bool, err error))
}

type callbacks struct {
	onSuccess func(role.ID)
	onFailure func(error)
}

type Manager struct {
	providersByDomain  map[string]Provider
	rolesByDomain      map[string]role.ID
	callbacksByRequest map[uint32]callbacks
}

func NewManager(providers map[string]Provider, roles map[string]role.ID) *Manager {
	return &Manager{
		providersByDomain:  providers,
		rolesByDomain:      roles,
		callbacksByRequest: map[uint32]callbacks{},
	}
}

func (m *Manager) TryAuthentication(domain, name string, onChal func(reqID uint32, chal string), onSuccess func(role.ID), onFailure func(error)) {
	p, ok := m.providersByDomain[domain]
	if !ok {
		onFailure(fmt.Errorf("auth: no provider for domain '%s'", domain))
		return
	}

	p.GenerateChallenge(name, func(reqID uint32, chal string, err error) {
		if err != nil {
			onFailure(err)
			return
		}
		m.callbacksByRequest[reqID] = callbacks{
			onSuccess: onSuccess,
			onFailure: onFailure,
		}
		onChal(reqID, chal)
	})

	return
}

func (m *Manager) CheckAnswer(reqID uint32, domain string, answ string) (err error) {
	defer delete(m.callbacksByRequest, reqID)

	p, ok := m.providersByDomain[domain]
	if !ok {
		err = fmt.Errorf("auth: no provider for domain '%s'", domain)
		return
	}

	callbacks, ok := m.callbacksByRequest[reqID]
	if !ok {
		err = fmt.Errorf("auth: unkown request '%d'", reqID)
		return
	}

	p.ConfirmAnswer(reqID, answ, func(ok bool, err error) {
		if err != nil {
			go callbacks.onFailure(err)
			return
		}
		go callbacks.onSuccess(m.rolesByDomain[domain])
	})

	return
}
