package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Notification holds the schema definition for the Notification entity.
type Notification struct {
	ent.Schema
}

// Fields of the Notification.
func (Notification) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			Immutable(),
		field.String("team_id").
			NotEmpty(),
		field.String("watch_condition_id").
			NotEmpty(),
		field.String("slot_id").
			NotEmpty(),
		field.String("channel").
			NotEmpty(),
		field.Enum("status").
			Values("pending", "sent", "failed").
			Default("pending"),
		field.Time("sent_at").
			Optional().
			Nillable(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the Notification.
func (Notification) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("team", Team.Type).
			Ref("notifications").
			Field("team_id").
			Unique().
			Required(),
		edge.From("watch_condition", WatchCondition.Type).
			Ref("notifications").
			Field("watch_condition_id").
			Unique().
			Required(),
		edge.From("slot", Slot.Type).
			Ref("notifications").
			Field("slot_id").
			Unique().
			Required(),
	}
}

// Indexes of the Notification.
func (Notification) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("team_id"),
		index.Fields("status"),
	}
}
