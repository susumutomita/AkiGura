package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Team holds the schema definition for the Team entity.
type Team struct {
	ent.Schema
}

// Fields of the Team.
func (Team) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			Immutable(),
		field.String("name").
			NotEmpty(),
		field.String("email").
			Unique().
			NotEmpty(),
		field.Enum("plan").
			Values("free", "personal", "pro", "org").
			Default("free"),
		field.Enum("status").
			Values("active", "paused", "cancelled").
			Default("active"),
		field.String("stripe_customer_id").
			Optional().
			Nillable(),
		field.String("stripe_subscription_id").
			Optional().
			Nillable(),
		field.Enum("billing_interval").
			Values("monthly", "yearly").
			Default("monthly").
			Optional(),
		field.Time("current_period_end").
			Optional().
			Nillable(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the Team.
func (Team) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("watch_conditions", WatchCondition.Type),
		edge.To("notifications", Notification.Type),
		edge.To("auth_tokens", AuthToken.Type),
		edge.To("support_tickets", SupportTicket.Type),
		edge.To("promo_code_usages", PromoCodeUsage.Type),
	}
}

// Indexes of the Team.
func (Team) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("stripe_customer_id"),
	}
}
