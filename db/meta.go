package db

import "strconv"

func (db *DB) getMeta() (map[string]string, error) {
	rows, err := db.DB.Query("SELECT key,value FROM meta;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return m, err
		}
		m[key] = value
	}
	return m, nil
}

type Meta struct {
	SchemaVersion   uint64
	BeerWeightGrams uint64
}

func (db *DB) GetMeta() (Meta, error) {
	m, err := db.getMeta()
	if err != nil {
		return Meta{}, err
	}

	schemaVersion, err := strconv.ParseUint(m["schema_version"], 10, 64)
	if err != nil {
		return Meta{}, err
	}

	beerWeight, err := strconv.ParseUint(m["beer_weight_grams"], 10, 64)
	if err != nil {
		return Meta{}, err
	}

	return Meta{
		SchemaVersion:   schemaVersion,
		BeerWeightGrams: beerWeight,
	}, nil
}
