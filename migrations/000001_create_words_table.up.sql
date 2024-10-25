CREATE TABLE IF NOT EXISTS words (
    id bigserial PRIMARY KEY,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    text_value text NOT NULL,
    difficulty text NOT NULL,
    related_words text[] NOT NULL,
    user_id bigserial NOT NULL,
    version integer NOT NULL DEFAULT 1
);

--     ID           int64     `json:"id"`
--     CreatedAt    time.Time `json:"-"`
--     Text         string    `json:"text"`
--     Difficulty   string    `json:"difficulty"`
--     RelatedWords []string  `json:"related_words,omitempty"`
--     UserId       int64     `json:"user_id,omitempty"`