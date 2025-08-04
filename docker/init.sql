create table users(
	login text primary key constraint unique_login unique,
	password text not null
	);



create table sessions(
    session_id UUID primary key default gen_random_uuid(),
    login text not null references users(login) on delete cascade
	);


CREATE TABLE docs (
    id UUID primary key default gen_random_uuid(),
    name text not null,
    mime text,
    file boolean not null,
    public boolean not null,
    created timestamp default NOW(),
    owner_login text not null references users(login),
    grant_logins text[],
    json_data JSONB
);