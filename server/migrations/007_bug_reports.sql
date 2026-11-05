-- Bug reports sent from the bug button at the foot of the site and on the
-- match page. Read them with `make bugs`.
CREATE TABLE bug_reports (
	id          bigserial PRIMARY KEY,
	category    text NOT NULL, -- ui, rules, connection, clock, bots, account, other
	match_id    text NOT NULL DEFAULT '', -- the match code, if the reporter gave one
	body        text NOT NULL,
	page        text NOT NULL DEFAULT '', -- the path the report was sent from
	user_id     bigint REFERENCES users (id) ON DELETE SET NULL,
	player      text NOT NULL DEFAULT '', -- the guest's browser ID when not signed in
	user_agent  text NOT NULL DEFAULT '',
	version     text NOT NULL DEFAULT '',
	created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX bug_reports_created_at ON bug_reports (created_at DESC);
