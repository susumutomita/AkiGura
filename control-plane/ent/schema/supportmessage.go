package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// SupportMessage holds the schema definition for the SupportMessage entity.
type SupportMessage struct {
	ent.Schema
}

// Fields of the SupportMessage.
func (SupportMessage) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			Immutable(),
		field.String("ticket_id").
			NotEmpty(),
		field.Enum("role").
			Values("user", "assistant", "system"),
		field.Text("content").
			NotEmpty(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the SupportMessage.
func (SupportMessage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ticket", SupportTicket.Type).
			Ref("messages").
			Field("ticket_id").
			Unique().
			Required(),
	}
}
