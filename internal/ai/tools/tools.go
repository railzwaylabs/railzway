package tools

import (
	genkitai "github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	customerdomain "github.com/railzwaylabs/railzway/internal/customer/domain"
	featuredomain "github.com/railzwaylabs/railzway/internal/feature/domain"
	invoicedomain "github.com/railzwaylabs/railzway/internal/invoice/domain"
	organizationdomain "github.com/railzwaylabs/railzway/internal/organization/domain"
	plandomain "github.com/railzwaylabs/railzway/internal/plan/domain"
	productdomain "github.com/railzwaylabs/railzway/internal/product/domain"
	subscriptiondomain "github.com/railzwaylabs/railzway/internal/subscription/domain"
	usagedomain "github.com/railzwaylabs/railzway/internal/usage/domain"
	aischeduler "github.com/railzwaylabs/railzway/internal/ai/scheduler"
	"go.uber.org/fx"
)

type Params struct {
	fx.In
	Genkit *genkit.Genkit

	Customer      customerdomain.Service
	Product       productdomain.Service
	Feature       featuredomain.Service
	Plan          plandomain.Service
	Subscription  subscriptiondomain.Service
	Invoice       invoicedomain.Service
	Usage         usagedomain.Service
	Organizations organizationdomain.Service
	Scheduler     *aischeduler.Service
}

func NewAssistantToolset(p Params) *AssistantToolset {
	if p.Genkit == nil {
		return nil
	}

	refs := make([]genkitai.ToolRef, 0, 20)
	refs = append(refs, defineCustomerTools(p.Genkit, p.Customer)...)
	refs = append(refs, defineProductTools(p.Genkit, p.Product)...)
	refs = append(refs, defineFeatureTools(p.Genkit, p.Feature)...)
	refs = append(refs, definePlanTools(p.Genkit, p.Plan)...)
	refs = append(refs, defineSubscriptionTools(p.Genkit, p.Subscription)...)
	refs = append(refs, defineInvoiceTools(p.Genkit, p.Invoice)...)
	refs = append(refs, defineUsageTools(p.Genkit, p.Usage)...)
	refs = append(refs, defineOrganizationTools(p.Genkit, p.Organizations)...)
	refs = append(refs, defineRecommendationTools(p.Genkit, p.Customer, p.Subscription, p.Invoice, p.Usage)...)
	refs = append(refs, defineSchedulerTools(p.Genkit, p.Scheduler)...)

	return &AssistantToolset{refs: refs}
}
