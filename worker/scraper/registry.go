package scraper

// Registry holds all available scrapers.
type Registry struct {
	scrapers map[string]func() Scraper
}

// NewRegistry creates a new scraper registry with all available scrapers.
func NewRegistry() *Registry {
	r := &Registry{
		scrapers: make(map[string]func() Scraper),
	}

	// Register all scrapers
	r.Register("kanagawa", func() Scraper { return NewKanagawaScraper() })
	r.Register("hiratsuka", func() Scraper { return NewHiratsukaScraper() })
	r.Register("yokohama", func() Scraper { return NewYokohamaScraper() })

	return r
}

// Register adds a scraper to the registry.
func (r *Registry) Register(name string, factory func() Scraper) {
	r.scrapers[name] = factory
}

// Get returns a new scraper instance by name.
func (r *Registry) Get(name string) Scraper {
	factory, ok := r.scrapers[name]
	if !ok {
		return nil
	}
	return factory()
}

// Names returns all registered scraper names.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.scrapers))
	for name := range r.scrapers {
		names = append(names, name)
	}
	return names
}
