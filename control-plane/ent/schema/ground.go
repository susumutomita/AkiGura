package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Ground holds the schema definition for the Ground entity.
type Ground struct {
	ent.Schema
}

// Fields of the Ground.
func (Ground) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			Immutable(),
		field.String("municipality_id").
			NotEmpty(),
		field.String("name").
			NotEmpty(),
		field.String("court_pattern").
			Optional(),
		field.Bool("enabled").
			Default(true),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the Ground.
func (Ground) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("municipality", Municipality.Type).
			Ref("grounds").
			Field("municipality_id").
			Unique().
			Required(),
		edge.To("slots", Slot.Type),
	}
}

// Indexes of the Ground.
func (Ground) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("municipality_id"),
		index.Fields("name"),
	}
}
