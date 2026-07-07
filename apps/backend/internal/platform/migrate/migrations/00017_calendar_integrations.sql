-- +goose Up

-- Per-user external calendar OAuth integrations (Google Calendar for now).
-- One-way import only: OpenTickly reads events, never writes back. Soft
-- delete on disconnect preserves any time entries already created from a
-- calendar event.
create table calendar_integrations (
    id bigserial primary key,
    user_id bigint not null references identity_users (id) on delete cascade,
    provider text not null,
    email text not null default '',
    access_token text not null default '',
    refresh_token text not null default '',
    token_expires_at timestamptz,
    has_write_scope boolean not null default false,
    error_status text not null default '',
    scopes text[] not null default '{}',
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    deleted_at timestamptz
);

create unique index calendar_integrations_user_provider_key
    on calendar_integrations (user_id, provider)
    where deleted_at is null;

create table calendar_integration_calendars (
    id bigserial primary key,
    calendar_integration_id bigint not null references calendar_integrations (id) on delete cascade,
    external_id text not null,
    name text not null default '',
    background_color text not null default '',
    foreground_color text not null default '',
    selected boolean not null default false,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint calendar_integration_calendars_unique
        unique (calendar_integration_id, external_id)
);

create index calendar_integration_calendars_integration_id_idx
    on calendar_integration_calendars (calendar_integration_id);

-- +goose Down

drop table if exists calendar_integration_calendars;
drop table if exists calendar_integrations;
