package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Facility holds the schema definition for the Facility entity (legacy).
type Facility struct {
	ent.Schema
}

// Fields of the Facility.
func (Facility) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			Immutable(),
		field.String("name").
			NotEmpty(),
		field.String("municipality").
			NotEmpty(),
		field.String("scraper_type").
			NotEmpty(),
		field.String("url").
			NotEmpty(),
		field.Bool("enabled").
			Default(true),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the Facility.
func (Facility) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("watch_conditions", WatchCondition.Type),
		edge.To("slots", Slot.Type),
	}
}
