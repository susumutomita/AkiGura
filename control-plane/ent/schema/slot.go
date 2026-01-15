package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Slot holds the schema definition for the Slot entity.
type Slot struct {
	ent.Schema
}

// Fields of the Slot.
func (Slot) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			Immutable(),
		field.Time("slot_date").
			SchemaType(map[string]string{
				"sqlite3": "date",
			}),
		field.String("time_from").
			NotEmpty(),
		field.String("time_to").
			NotEmpty(),
		field.String("court_name").
			Optional(),
		field.String("raw_text").
			Optional(),
		field.Time("scraped_at").
			Default(time.Now),
	}
}

// Edges of the Slot.
func (Slot) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("facility", Facility.Type).
			Ref("slots").
			Unique(),
		edge.From("municipality", Municipality.Type).
			Ref("slots").
			Unique(),
		edge.From("ground", Ground.Type).
			Ref("slots").
			Unique(),
		edge.To("notifications", Notification.Type),
	}
}

// Indexes of the Slot.
func (Slot) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("slot_date"),
		index.Edges("municipality"),
		index.Edges("ground"),
	}
}
