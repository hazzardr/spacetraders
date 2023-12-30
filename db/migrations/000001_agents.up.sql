create schema spacetraders;

create table if not exists spacetraders.agents (
    id bigserial primary key, -- auto incrementing id
    call_sign text not null,
    faction text not null,
    headquarters text not null,
    credits integer not null,
    expires_on date not null
);

create table if not exists spacetraders.ships (
   id bigserial primary key, -- auto incrementing id
   type text not null,
   owner int references spacetraders.agents(id)
);
