package koding

import (
	"fmt"
	"koding/db/mongodb"

	"github.com/koding/kloud/protocol"
	"github.com/koding/kloud/provider/amazon"
	"github.com/koding/logging"
)

type CheckContext struct {
	api      *amazon.AmazonClient
	db       *mongodb.MongoDB
	username string
	env      string
	log      logging.Logger
}

// Limiter checks the limits via the Check() method. It should simply return an
// error if the limitations are exceed.
type Limiter interface {
	Check(ctx *CheckContext) error
}

type multiLimiter []Limiter

func (m multiLimiter) Check(ctx *CheckContext) error {
	for _, limiter := range m {
		if err := limiter.Check(ctx); err != nil {
			return err
		}
	}

	return nil
}

func newMultiLimiter(limiter ...Limiter) Limiter {
	return multiLimiter(limiter)
}

// Limit implements the kloud.Limiter interface. This is called for every
// incoming method before the execution.
func (p *Provider) Limit(opts *protocol.Machine, method string) error {
	// only check for build method, all other's are ok to be used without any
	// restriction.
	if method != "build" {
		return nil
	}

	a, err := p.NewClient(opts)
	if err != nil {
		return err
	}

	username := opts.Builder["username"].(string)

	ctx := &CheckContext{
		api:      a,
		db:       p.Session,
		username: username,
		env:      p.Kite.Config.Environment,
		log:      p.Log,
	}

	plan, err := p.getPlan()
	if err != nil {
		return err
	}

	return p.CheckLimits(plan, ctx)
}

func (p *Provider) getPlan() (Plan, error) {
	return Free, nil
}

// CheckLimits checks the given user limits
func (p *Provider) CheckLimits(plan Plan, ctx *CheckContext) error {
	l, ok := limits[plan]
	if !ok {
		fmt.Errorf("plan %s not found", plan)
	}

	return l.Check(ctx)
}
