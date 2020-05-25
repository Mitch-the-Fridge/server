package db

import "time"

type Grab struct {
	ID     int64
	ClipID string

	GrabberGuess   int64
	GuessCertainty float64
	DateGuessed    time.Time

	GrabberManual    *int64
	DateGrabberGiven *time.Time
}

func scanGrab(s interface {
	Scan(dest ...interface{}) error
}) (Grab, error) {
	var g Grab
	if err := s.Scan(&g.ID, &g.ClipID, &g.GrabberGuess, &g.GuessCertainty, &g.DateGuessed, &g.GrabberManual, &g.DateGrabberGiven); err != nil {
		return g, err
	}

	return g, nil
}

func (db *DB) GetGrabForClip(clipId string) (grab Grab, found bool, err error) {
	rows, err := db.DB.Query("SELECT * FROM grabs WHERE clip_id = ?", clipId)
	if err != nil {
		return Grab{}, false, err
	}
	defer rows.Close()

	if rows.Next() {
		grab, err := scanGrab(rows)
		return grab, true, err
	} else {
		return Grab{}, false, nil
	}
}
