create schema if not exists spacetraders;

create table if not exists spacetraders.agents (
    id bigserial primary key, -- auto incrementing id
    call_sign text not null,
    faction text not null,
    headquarters text not null,
    credits integer not null,
    expires_on date not null, --expiration date for the agent, typically every two weeks,
    email text not null -- email is used to preserve callsign across resets
);

create unique index if not exists agents_call_sign_idx on spacetraders.agents(call_sign);

create table if not exists spacetraders.ships (
   id bigserial primary key, -- auto incrementing id
   type text not null,
   owner int references spacetraders.agents(id)
);

create table if not exists spacetraders.auth (
    id bigserial primary key, -- auto incrementing id
    token bytea not null, -- hashed token from spacetraders
    expires_on date not null, --expiration for token
    agent_id int references spacetraders.agents(id)
);