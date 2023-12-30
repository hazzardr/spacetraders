create table ships (
   id bigserial primary key, -- auto incrementing id
   type text not null,
   owner text references agents(call_sign)
);

create table agents (
    id bigserial primary key, -- auto incrementing id
    call_sign text not null,
    faction text not null,
    headquarters text not null,
    credits integer not null
);

CREATE UNIQUE INDEX ships_type_owner_idx ON agents(call_sign);