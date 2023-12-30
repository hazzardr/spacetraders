-- name: GetAgentByCallsign :one
select *
from spacetraders.agents
where call_sign = $1;

-- name: InsertAgent :one
insert into spacetraders.agents (call_sign, faction, headquarters, credits, expires_on) values ($1, $2, $3, $4, $5) returning *;

-- name: GetShips :many
select * from spacetraders.ships;

-- name: InsertShip :one
insert into spacetraders.ships (type, owner) values ($1, $2) returning *;