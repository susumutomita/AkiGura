package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// WatchCondition holds the schema definition for the WatchCondition entity.
type WatchCondition struct {
	ent.Schema
}

// Fields of the WatchCondition.
func (WatchCondition) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			Immutable(),
		field.String("days_of_week").
			NotEmpty(), // JSON array like "[0,6]"
		field.String("time_from").
			NotEmpty(),
		field.String("time_to").
			NotEmpty(),
		field.Time("date_from").
			Optional().
			Nillable(),
		field.Time("date_to").
			Optional().
			Nillable(),
		field.Bool("enabled").
			Default(true),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the WatchCondition.
func (WatchCondition) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("team", Team.Type).
			Ref("watch_conditions").
			Unique().
			Required(),
		edge.From("facility", Facility.Type).
			Ref("watch_conditions").
			Unique().
			Required(),
		edge.To("notifications", Notification.Type),
	}
}

// Indexes of the WatchCondition.
func (WatchCondition) Indexes() []ent.Index {
	return []ent.Index{
		index.Edges("team"),
		index.Edges("facility"),
	}
}
