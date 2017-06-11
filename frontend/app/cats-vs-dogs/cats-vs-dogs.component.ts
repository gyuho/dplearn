import {
  Component,
  OnDestroy,
} from '@angular/core';

import {
  Http,
  Response,
  Headers,
  RequestOptions,
} from '@angular/http';

import {
  Observable,
} from 'rxjs/Rx';

import {
  MdSnackBar,
} from '@angular/material';

import {
  Request,
  Item,
} from '../request-item.component';

@Component({
  selector: 'app',
  templateUrl: 'cats-vs-dogs.component.html',
  styleUrls: ['cats-vs-dogs.component.css'],
})
export class CatsVsDogsComponent implements OnDestroy {
  private endpoint = 'cats-vs-dogs-request';

  mode = 'Observable';

  inputValue: string;

  sresp: Item;
  srespError: string;
  result: string;

  inProgress = false;
  spinnerColor = 'primary';
  spinnerMode = 'determinate';
  spinnerValue = 0;

  pollingHandler;

  constructor(private http: Http, public snackBar: MdSnackBar) {
    this.inputValue = 'https://images.pexels.com/photos/127028/pexels-photo-127028.jpeg';
    this.srespError = '';
    this.result = 'No results to show yet...';
  }

  ngOnDestroy() {
    console.log('Disconnected (user left the page)!');
    clearInterval(this.pollingHandler);

    // DELETE request to backend
    console.log('sending DELETE');
    let creq = new Request(this.inputValue, true);
    let responseFromSubscribe: Item;
    this.deleteRequest(creq).subscribe(
      sresp => responseFromSubscribe = sresp,
      error => this.srespError = <any>error,
      () => this.processItem(responseFromSubscribe), // on-complete
    );
    console.log('sent DELETE');

    this.inputValue = '';
    this.srespError = '';
    return;
  }

  processItem(resp: Item) {
    this.sresp = resp;
    this.result = resp.value;
    if (resp.error !== '') {
      clearInterval(this.pollingHandler);
      if (this.result !== '') {
        this.result = resp.value + '(' + resp.error + ')';
      } else {
        this.result = resp.error;
      }
    }
    if (resp.canceled === true) {
      this.result += ' - canceled!';
    }

    this.inProgress = resp.progress < 100;
    this.spinnerValue = resp.progress;

    if (resp.progress === 100) {
      console.log('Finished', resp);
      clearInterval(this.pollingHandler);
    }
  }

  processHTTPResponseClient(res: Response) {
    let jsonBody = res.json();
    let sresp = <Item>jsonBody;
    return sresp || {};
  }

  processHTTPErrorClient(error: any) {
    let errMsg = (error.message) ? error.message :
      error.status ? `${error.status} - ${error.statusText}` : 'Server error';
    console.error(errMsg);
    this.srespError = errMsg;
    return Observable.throw(errMsg);
  }

  postRequest(creq: Request): Observable<Item> {
    let body = JSON.stringify(creq);
    let headers = new Headers({'Content-Type' : 'application/json'});
    let options = new RequestOptions({headers : headers});

    // this returns without waiting for POST response
    let obser = this.http.post(this.endpoint, body, options)
      .map(this.processHTTPResponseClient)
      .catch(this.processHTTPErrorClient);
    return obser;
  }

  deleteRequest(creq: Request): Observable<Item> {
    creq.delete_request = true;
    return this.postRequest(creq);
  }

  processRequest() {
    let creq = new Request(this.inputValue, false);
    let responseFromSubscribe: Item;
    this.postRequest(creq).subscribe(
      sresp => responseFromSubscribe = sresp,
      error => this.srespError = <any>error,
      () => this.processItem(responseFromSubscribe), // on-complete
    );
  }

  clickProcessRequest() {
    this.snackBar.open('Predicting correct words...', 'Requested!', {
      duration: 5000,
    });
    this.inProgress = true;
    this.spinnerMode = 'determinate';
    this.spinnerValue = 0;
    this.pollingHandler = setInterval(() => this.processRequest(), 1000);
  }
}
