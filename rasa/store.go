package rasa

type Intent struct {
	Name     string
	Examples []string
}

type Store struct {
	intents map[string]Intent
}

func NewStore() *Store {
	return &Store{
		intents: make(map[string]Intent),
	}
}

func (s *Store) Upsert(data map[string]interface{}) {
	name := data["intent"].(string)
	examplesIface := data["examples"].([]interface{})
	examples := []string{}
	for _, e := range examplesIface {
		examples = append(examples, e.(string))
	}

	s.intents[name] = Intent{
		Name:     name,
		Examples: examples,
	}
}

func (s *Store) All() []Intent {
	out := []Intent{}
	for _, v := range s.intents {
		out = append(out, v)
	}
	return out
}
