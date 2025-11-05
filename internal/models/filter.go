package models

// SavedFilter represents a saved bulk operation filter
type SavedFilter struct {
	Name            string `yaml:"name"`
	NamePattern     string `yaml:"name_pattern"`
	AgeOp           string `yaml:"age_op"`
	AgeValue        int    `yaml:"age_value"`
	AgeUnit         string `yaml:"age_unit"`
	ConsumerOp      string `yaml:"consumer_op"`
	ConsumerValue   int    `yaml:"consumer_value"`
	MessagesOp      string `yaml:"messages_op"`
	MessagesValue   int64  `yaml:"messages_value"`
}

// FilterConfig holds all saved filters
type FilterConfig struct {
	Filters []SavedFilter `yaml:"filters"`
}

