package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ScrapeJob holds the schema definition for the ScrapeJob entity.
type ScrapeJob struct {
	ent.Schema
}

// Fields of the ScrapeJob.
func (ScrapeJob) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			Immutable(),
		field.Enum("status").
			Values("pending", "running", "completed", "failed").
			Default("pending"),
		field.Int("slots_found").
			Default(0),
		field.String("error_message").
			Optional().
			Nillable(),
		field.String("scrape_status").
			Optional().
			Nillable(),
		field.Text("diagnostics").
			Optional().
			Nillable(),
		field.Time("started_at").
			Optional().
			Nillable(),
		field.Time("completed_at").
			Optional().
			Nillable(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the ScrapeJob.
func (ScrapeJob) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("municipality", Municipality.Type).
			Ref("scrape_jobs").
			Unique().
			Required(),
	}
}

// Indexes of the ScrapeJob.
func (ScrapeJob) Indexes() []ent.Index {
	return []ent.Index{
		index.Edges("municipality"),
		index.Fields("status"),
	}
}
