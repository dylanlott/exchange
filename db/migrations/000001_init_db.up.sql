CREATE TABLE campaigns (
	pk bigserial NOT NULL,
	id text NOT NULL,
	created_at timestamp with time zone NOT NULL,
	updated_at timestamp with time zone NOT NULL,
	industry text NOT NULL,
	PRIMARY KEY ( pk ),
	UNIQUE ( id )
);
CREATE TABLE leads (
	pk bigserial NOT NULL,
	created_at timestamp with time zone NOT NULL,
	updated_at timestamp with time zone NOT NULL,
	industry text NOT NULL,
	PRIMARY KEY ( pk )
);
CREATE TABLE marketings (
	pk bigserial NOT NULL,
	email text NOT NULL,
	created_at timestamp with time zone NOT NULL,
	PRIMARY KEY ( pk ),
	UNIQUE ( email )
);
CREATE TABLE users (
	pk bigserial NOT NULL,
	created_at timestamp with time zone NOT NULL,
	updated_at timestamp with time zone NOT NULL,
	id text NOT NULL,
	name text NOT NULL,
	email text NOT NULL,
	password text NOT NULL,
	role text NOT NULL,
	industry text NOT NULL,
	PRIMARY KEY ( pk ),
	UNIQUE ( id ),
	UNIQUE ( email )
);
