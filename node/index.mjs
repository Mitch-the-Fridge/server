import * as fs from 'fs/promises';
import { createInterface } from 'readline';
import _ from 'lodash';

import Database from './db.mjs';

import tf from '@tensorflow/tfjs-node';
import knnClassifier from '@tensorflow-models/knn-classifier';
import canvas from 'canvas';
import faceapi from 'face-api.js';

const { Canvas, Image, ImageData, loadImage } = canvas;
faceapi.env.monkeyPatch({ Canvas, Image, ImageData });

function log(...args) {
	console.error('[node]', ...args);
}

const db = new Database(process.argv[2]);

/*
async function handleDirectory(clip) {
	let [files, accounts] = await Promise.all([ fs.readdir(clip.dir), db.getAllAccounts() ]);
	files = files.map(p => `${clip.dir}/${p}`);

	const IDs = accounts.map(a => a.id);
	const encodings = accounts.map(a => JSON.parse(a.face_encoding));

	// returns row-major matrix where columns are accounts and rows are frames,
	// in the same order as IDs and encodings.
	let m = await Promise.all(files.map(f => handleFrame(encodings, f)));

	// remove empty rows
	m = _.compact(m);

	const height = m.length;
	const width = m[0].length;

	// normalize every row using 2-norm
	for (let i = 0; i < height; i++) {
		//m[i] = normalizeVector(m[i]);
		// REVIEW: normalization disabled, is this needed?
	}

	// sum over the rows of the matrix to create a vector.
	let v = new Array(width);
	for (let x = 0; x < width; x++) {
		// REVIEW: average instead of sum?
		//v[x] = _.chain(m).map(v => v[x]).sum().value();

		const sum = _.chain(m).map(v => v[x]).sum().value();
		v[x] = sum / height;
	}

	// normalize the vector
	//v = normalizeVector(v);
	log('v=', v);

	const [userId, certainty] = _.chain(IDs)
		.zip(v)
		.maxBy(([_, certainty]) => certainty)
		.value();

	await db.insertGrab(clip.id, userId, certainty);
	await fs.rmdir(clip.dir, { recursive: true });
}

async function handleFrame(encodings, fname) {
	const image = await loadImage(fname);
	const face = await faceapi.detectSingleFace(image).withFaceLandmarks().withFaceDescriptor();
	if (face == null) {
		return null;
	}

	log(JSON.stringify(face.descriptor));

	return encodings.map(e => {
		const dist = faceapi.euclideanDistance(e, face.descriptor);
		return (1 - (dist/2)) ** 2;
	})
}
*/

async function handleDirectory(knn, clip) {
	let files = await fs.readdir(clip.dir);
	files = files.map(p => `${clip.dir}/${p}`);

	// returns array of a vector embedding for every frame.
	let m = await Promise.all(files.map(handleFrame));
	const oldLength = m.length;

	// remove empty rows
	m = _.compact(m);
	const ratAfterCompact = m.length / oldLength;

	const width = m[0].length;

	// v is the average face embedding
	let v = new Array(width);
	for (let x = 0; x < width; x++) {
		v[x] = _.chain(m).map(v => v[x]).mean().value();
	}
	log("v=", v);

	const {label: userId, confidences} = await knn.predictClass(tf.tensor(v));

	log("clip", clip.id, userId, confidences, "rat after compact", ratAfterCompact);
	await db.insertGrab(clip.id, userId, confidences[userId]);

	// signal to the Go process that we're done processing this clip
	console.log(clip.id);

	await fs.rmdir(clip.dir, { recursive: true });
}

async function handleFrame(fname) {
	const image = await loadImage(fname);
	const face = await faceapi.detectSingleFace(image).withFaceLandmarks().withFaceDescriptor();
	if (face == null) {
		return null;
	}

	return face.descriptor;
}

function normalizeVector(v) {
	const length = Math.sqrt(_.chain(v).map(x => x**2).sum().value());
	return v.map(x => x / length);
}

async function updateClassifier(knn, obj) {
	knn.addExample(tf.tensor(obj.encoding), obj.user_id);

	let model = knn.getClassifierDataset();
	model = _.chain(model)
		.toPairs()
		.map(([key, val]) => [key, val.arraySync()])
		.fromPairs()
		.value();
	await db.setKNNModel(model);
}

(async function() {
	await Promise.all([
		faceapi.nets.ssdMobilenetv1.loadFromDisk('./face-api.js/weights'),
		faceapi.nets.faceRecognitionNet.loadFromDisk('./face-api.js/weights'),
		faceapi.nets.faceLandmark68Net.loadFromDisk('./face-api.js/weights'),
	]);

	const knn = knnClassifier.create();
	const model = _.chain(await db.getKNNModel())
		.toPairs()
		.map(([key, val]) => [key, tf.tensor2d(val)])
		.fromPairs()
		.value();
	log("got knn model from db", model);
	if (!_.isEmpty(model)) {
		knn.setClassifierDataset(model);
	}

	const rl = createInterface({
		input: process.stdin,
		output: process.stdout,
		terminal: false,
	});

	rl.on('line', str => {
		const obj = JSON.parse(str.slice(1));

		switch (str[0]) {
		case 'W':
			log('working on', obj);
			handleDirectory(knn, obj);
			break;
		case 'E':
			log('train classifier request', obj);
			updateClassifier(knn, obj);
			break;
		}
	});

	return 0;
})().catch(e => {
	console.error('[node]', e);
});
