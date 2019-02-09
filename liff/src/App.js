import React, { Component } from 'react';
import './App.css';
import man from './assets/img/man.png';
import liffUtil from './utils/liffUtil';
import swal from 'sweetalert2';

class App extends Component {
  constructor(props) {
    super(props);
    this.state = {
      profile: {
        pictureUrl: man,
      }
    };
  }

  async componentWillMount() {
    const profile = await liffUtil.getProfile();
    this.setState({ profile });
  }

  render() {
    return (
      <div className="app">
        <header className="app-header">
          <h3 className="app-title">しゃべらせるで！！
            <img className="avatar-img" alt="profile" src={this.state.profile.pictureUrl} />
          </h3>
        </header>
        <div className="page-content"></div>
        <button type="button" className="btn btn-default btn-block message-button" onClick={() => this.sendTextMessage('帰るねー') }>
          帰るねー
        </button>
        <div className="page-content"></div>
        <button type="button" className="btn btn-default btn-block message-button" onClick={() => this.sendTextMessage('帰りまっせ') }>
          帰りまっせ
        </button>
        <div className="page-content"></div>
        <button type="button" className="btn btn-default btn-block message-button" onClick={() => this.sendTextMessage('腹ペコ') }>
          腹ペコ
        </button>
      </div>
    );
  }

  async sendTextMessage(text) {
    if (!text) {
      return;
    }

    const message = {
      type: 'text',
      text: text
    };

    try {
      await liffUtil.sendMessages([message]);
      swal({
        type: 'success',
        title: 'Send Message Complete',
        showConfirmButton: false,
        timer: 1000
      });

    } catch (err) {
      swal({
        type: 'error',
        title: 'Send Error',
        text: err.response.data.message,
      });
    }
  }
}

export default App;
