package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// SupportTicket holds the schema definition for the SupportTicket entity.
type SupportTicket struct {
	ent.Schema
}

// Fields of the SupportTicket.
func (SupportTicket) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			Immutable(),
		field.String("email").
			NotEmpty(),
		field.String("subject").
			NotEmpty(),
		field.Enum("status").
			Values("open", "ai_handled", "escalated", "resolved", "closed").
			Default("open"),
		field.Enum("priority").
			Values("low", "normal", "high", "urgent").
			Default("normal"),
		field.Text("ai_response").
			Optional().
			Nillable(),
		field.Text("human_response").
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

// Edges of the SupportTicket.
func (SupportTicket) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("team", Team.Type).
			Ref("support_tickets").
			Unique(),
		edge.To("messages", SupportMessage.Type),
	}
}

// Indexes of the SupportTicket.
func (SupportTicket) Indexes() []ent.Index {
	return []ent.Index{
		index.Edges("team"),
		index.Fields("status"),
	}
}
