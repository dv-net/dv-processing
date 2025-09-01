create table if not exists settings
(
    id         uuid primary key                  default gen_random_uuid() not null,
    model_id   uuid,
    model_type varchar(255),
    name       varchar(255)             not null,
    value      varchar(255)             not null,
    created_at timestamp with time zone not null default now(),
    updated_at timestamp with time zone,
    UNIQUE (model_id, model_type, name)
);

-- Insert processing uuid via service initialization
insert into settings (name, value) values ('X-Processing-ID', gen_random_uuid());