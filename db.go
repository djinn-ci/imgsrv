package main

import (
	_ "embed"
	"time"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"

	"github.com/andrewpillar/query"
)

//go:embed schema.sql
var schema string

type DB struct {
	*sqlite.Conn
}

func InitDB() (DB, error) {
	var db DB

	conn, err := sqlite.OpenConn(":memory:", sqlite.SQLITE_OPEN_READWRITE)

	if err != nil {
		return db, err
	}

	if err := sqlitex.ExecScript(conn, schema); err != nil {
		return db, err
	}

	return DB{
		Conn: conn,
	}, nil
}

var (
	insertImg = `
INSERT INTO images
(path, driver, category, group_name, name, link, mod_time)
VALUES ($1, $2, $3, $4, $5, $6, $7)
`

	updateImg = `
UPDATE images
SET mod_time = $1
WHERE (path = $2)
`
)

func (db DB) Load(imgs []*Image) error {
	for _, img := range imgs {
		stmt, err := db.Prepare(insertImg)

		if err != nil {
			return err
		}

		stmt.BindText(1, img.Path)
		stmt.BindText(2, img.Driver)
		stmt.BindText(3, img.Category)
		stmt.BindText(4, img.Group)
		stmt.BindText(5, img.Name)
		stmt.BindText(6, img.Link)
		stmt.BindInt64(7, img.ModTime.Unix())

		if _, err := stmt.Step(); err != nil {
			sqlerr, _ := err.(sqlite.Error)

			if sqlerr.Code == sqlite.SQLITE_CONSTRAINT_UNIQUE {
				stmt, err := db.Prepare(updateImg)

				if err != nil {
					return err
				}

				stmt.BindInt64(1, img.ModTime.Unix())
				stmt.BindText(2, img.Path)

				if _, err := stmt.Step(); err != nil {
					return err
				}

				if err := stmt.ClearBindings(); err != nil {
					return err
				}
				continue
			}
		}

		if err := stmt.ClearBindings(); err != nil {
			return err
		}
	}
	return nil
}

func (db DB) Sync(imgs []*Image) (int, error) {
	nop := func(_ *sqlite.Stmt) error { return nil }

	paths := make([]interface{}, 0, len(imgs))

	for _, img := range imgs {
		paths = append(paths, img.Path)
	}

	q := query.Delete("images", query.Where("path", "NOT IN", query.List(paths...)))

	if err := sqlitex.Exec(db.Conn, q.Build(), nop, q.Args()...); err != nil {
		return 0, err
	}

	set := make(map[string]int64)

	scan := func(stmt *sqlite.Stmt) error {
		set[stmt.ColumnText(0)] = stmt.ColumnInt64(1)

		return nil
	}

	if err := sqlitex.Exec(db.Conn, "SELECT path, mod_time FROM images", scan); err != nil {
		return 0, err
	}

	new := make([]*Image, 0, len(imgs))

	for _, img := range imgs {
		if mod, ok := set[img.Path];ok {
			if mod == img.ModTime.Unix() {
				continue
			}
		}
		new = append(new, img)
	}

	if err := db.Load(new); err != nil {
		return 0, err
	}
	return len(new), nil
}

func WhereDriver(driver string) query.Option {
	return func(q query.Query) query.Query {
		if driver == "" {
			return q
		}
		return query.Where("driver", "=", query.Arg(driver))(q)
	}
}

func WhereCategory(category string) query.Option {
	return func(q query.Query) query.Query {
		if category == "" {
			return q
		}
		return query.Where("category", "=", query.Arg(category))(q)
	}
}

var imageCols = []string{
	"path",
	"driver",
	"category",
	"group_name",
	"name",
	"link",
	"mod_time",
}

func scanImage(img *Image) func(*sqlite.Stmt) error {
	return func(stmt *sqlite.Stmt) error {
		modtime := stmt.ColumnInt64(6)

		img.Path = stmt.ColumnText(0)
		img.Driver = stmt.ColumnText(1)
		img.Category = stmt.ColumnText(2)
		img.Group = stmt.ColumnText(3)
		img.Name = stmt.ColumnText(4)
		img.Link = stmt.ColumnText(5)
		img.ModTime = time.Unix(modtime, 0)
		return nil
	}
}

func (db DB) Image(driver, category, name string) (*Image, bool, error) {
	q := query.Select(
		query.Columns(imageCols...),
		query.From("images"),
		query.Where("driver", "=", query.Arg(driver)),
		query.Where("category", "=", query.Arg(category)),
		query.Where("name", "=", query.Arg(name)),
	)

	var img Image

	if err := sqlitex.Exec(db.Conn, q.Build(), scanImage(&img), q.Args()...); err != nil {
		if err, ok := err.(sqlite.Error); ok && err.Code == sqlite.SQLITE_NOTFOUND {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &img, img.Path != "", nil
}

func (db DB) Images(opts ...query.Option) ([]*Image, error) {
	opts = append([]query.Option{
		query.From("images"),
	}, opts...)

	q := query.Select(query.Columns(imageCols...), opts...)

	imgs := make([]*Image, 0)

	set := make(map[string]*Image)
	paths := make([]interface{}, 0)

	scan := func(stmt *sqlite.Stmt) error {
		img := &Image{}

		if err := scanImage(img)(stmt); err != nil {
			return err
		}

		set[img.Path] = img

		paths = append(paths, img.Path)
		imgs = append(imgs, img)

		return nil
	}

	if err := sqlitex.Exec(db.Conn, q.Build(), scan, q.Args()...); err != nil {
		return nil, err
	}
	return imgs, nil
}
