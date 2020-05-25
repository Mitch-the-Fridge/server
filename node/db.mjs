import sqlite3 from 'sqlite3';

function log(...args) {
	console.error('[node]', ...args);
}

export default class Database {
	constructor(fname) {
		log('opening', fname);
		this.db = new sqlite3.Database(fname);
	}

	async _allQuery(query) {
		return await new Promise((resolve, reject) => {
			this.db.all(query, (err, rows) => {
				if (err != null) {
					reject(err);
				} else {
					resolve(rows);
				}
			})
		});
	}

	async getMeta() {
		const rows = await this._allQuery("select * from meta;");

		let res = {};
		for (const row of rows) {
			res[row.key] = row.value;
		}
		return res;
	}

	async getAllAccounts() {
		return await this._allQuery("select id,face_encoding from persons;");
	}

	async insertGrab(clipId, userId, certainty) {
		return await new Promise((resolve, reject) => {
			this.db.run(
				"INSERT INTO grabs(clip_id, grabber_guess, guess_certainty, date_guessed) values(?, ?, ?, ?)",
				clipId,
				userId,
				certainty,
				new Date(),
				err => {
					if (err != null) {
						reject(err);
					} else {
						resolve();
					}
				}
			)
		});
	}

	async getKNNModel() {
		const meta = await this.getMeta();

		if (meta['knn'] == null) {
			return null;
		}

		return JSON.parse(meta['knn']);
	}

	async setKNNModel(model) {
		const json = JSON.stringify(model);

		return await new Promise((resolve, reject) => {
			this.db.run(
				"REPLACE INTO meta(key, value) VALUES(?, ?)",
				'knn',
				json,
				err => {
					if (err != null) {
						reject(err);
					} else {
						resolve();
					}
				}
			)
		});
	}
}
