

CREATE TABLE images (
	path       VARCHAR NOT NULL UNIQUE,
	driver     VARCHAR NOT NULL,
	category   VARCHAR NULL,
	group_name VARCHAR NULL,
	name       VARCHAR NOT NULL,
	link       VARCHAR NOT NULL,
	mod_time   INT NOT NULL
);
