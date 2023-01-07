package memory

type LanguageRepository struct {
	m map[int]string
}

func NewLanguageRepository() *LanguageRepository {
	return &LanguageRepository{m: make(map[int]string)}
}

func (l *LanguageRepository) GetLang(id int) (string, error) {
	return l.m[id], nil
}

func (l *LanguageRepository) SetLang(id int, lang string) error {
	l.m[id] = lang
	return nil
}
