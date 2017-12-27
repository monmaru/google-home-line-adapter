const request = require('superagent');
const firebase = require('firebase-admin');
// your settings
const serviceAccount = require('./serviceAccountKey.json');
const googleHomeNotifierURL = '';
const firebaseURL = '';

const sendMessage = (message) => {
  request.post(googleHomeNotifierURL)
    .type('form')
    .send({
      text: message
    })
    .end((err, res) => {
      if (err) {
        console.error(err);
      } else {
        console.log(res.text);
      }
    });
  }

firebase.initializeApp({
  credential: firebase.credential.cert(serviceAccount),
  databaseURL: firebaseURL
});

firebase.database().ref('/linebot/receive').on('value', (changedSnapshot) => {
  const message = changedSnapshot.child('message').val();
  if (message) {
    sendMessage(message);
  }}, (errorObject) => {
    console.log(`The read failed: ${errorObject.code}`);
  });
