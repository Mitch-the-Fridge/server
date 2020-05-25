package db

import (
	"database/sql"
	"time"
)

type Clip struct {
	ID          string    `json:"id"`
	FPS         int       `json:"fps"`
	FrameCount  int64     `json:"frame_count"`
	Start       time.Time `json:"start_date"`
	End         time.Time `json:"end_date"`
	BeginWeight float64   `json:"begin_weight"`
	EndWeight   float64   `json:"end_weight"`
}

func scanClip(s interface {
	Scan(dest ...interface{}) error
}) (Clip, error) {
	var c Clip
	if err := s.Scan(&c.ID, &c.FPS, &c.FrameCount, &c.Start, &c.BeginWeight, &c.EndWeight); err != nil {
		return c, err
	}

	secs := float64(c.FrameCount) / float64(c.FPS)
	millis := secs * 1000
	c.End = c.Start.Add(time.Duration(millis) * time.Millisecond)

	return c, nil
}

func (db *DB) getClips() ([]Clip, error) {
	var res []Clip

	rows, err := db.DB.Query("SELECT id, fps, frame_count, start_date FROM clips;")
	if err != nil {
		return res, err
	}
	defer rows.Close()

	for rows.Next() {
		clip, err := scanClip(rows)
		if err != nil {
			return res, err
		}
		res = append(res, clip)
	}

	return res, nil
}

func (db *DB) GetClip(id string) (Clip, bool, error) {
	rows, err := db.DB.Query("SELECT id, fps, frame_count, start_date FROM clips WHERE id = ?;", id)
	if err != nil {
		return Clip{}, false, err
	}
	defer rows.Close()

	if rows.Next() {
		clip, err := scanClip(rows)
		return clip, true, err
	} else {
		return Clip{}, false, nil
	}
}

func (db *DB) InsertClip(clip Clip) (sql.Result, error) {
	return db.DB.Exec(
		"insert into clips(id, fps, frame_count, start_date, begin_weight, end_weight) values (?, ?, ?, ?, ?, ?)",
		clip.ID,
		clip.FPS,
		clip.FrameCount,
		clip.Start,
		clip.BeginWeight,
		clip.EndWeight,
	)
}
