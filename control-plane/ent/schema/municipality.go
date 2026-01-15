package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Municipality holds the schema definition for the Municipality entity.
type Municipality struct {
	ent.Schema
}

// Fields of the Municipality.
func (Municipality) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			Immutable(),
		field.String("name").
			NotEmpty(),
		field.String("scraper_type").
			Unique().
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

// Edges of the Municipality.
func (Municipality) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("grounds", Ground.Type),
		edge.To("slots", Slot.Type),
		edge.To("scrape_jobs", ScrapeJob.Type),
	}
}

// Indexes of the Municipality.
func (Municipality) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("scraper_type").Unique(),
	}
}
