import {
  Component,
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
  selector: 'app-word-predict',
  templateUrl: 'word-predict.component.html',
  styleUrls: ['word-predict.component.css'],
})
export class WordPredictComponent {
  mode = 'Observable';
  private endpointI = 'word-predict-request-1';
  private endpointII = 'word-predict-request-2';

  inputValueI: string;
  inputValueII: string;

  sresp: Item;
  srespError: string;

  resultI: string;
  resultII: string;

  inProgressI = false;
  spinnerColorI = 'primary';
  spinnerModeI = 'determinate';
  spinnerValueI = 0;

  inProgressII = false;
  spinnerColorII = 'primary';
  spinnerModeII = 'determinate';
  spinnerValueII = 0;

  constructor(private http: Http, public snackBar: MdSnackBar) {
    this.inputValueI = '';
    this.inputValueII = '';
    this.srespError = '';
    this.resultI = 'No results to show yet...';
    this.resultII = 'No results to show yet...';
  }

  // ngOnInit(): void {}
  // ngAfterContentInit() {}
  // ngAfterViewInit() {}
  // ngAfterViewChecked() {}
  // ngOnDestroy() {
  //   console.log('Disconnected from cluster (user left the page)!');
  //   return;
  // }

  processItemI(resp: Item) {
    this.sresp = resp;
    this.resultI = resp.value;
    this.inProgressI = resp.progress < 100;
    this.spinnerModeI = 'determinate';
    this.spinnerValueI = resp.progress;
  }
  processItemII(resp: Item) {
    this.sresp = resp;
    this.resultII = resp.value;
    this.inProgressII = resp.progress < 100;
    this.spinnerModeII = 'determinate';
    this.spinnerValueII = resp.progress;
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

  postRequestI(creq: Request): Observable<Item> {
    let body = JSON.stringify(creq);
    let headers = new Headers({'Content-Type' : 'application/json'});
    let options = new RequestOptions({headers : headers});

    // this returns without waiting for POST response
    let obser = this.http.post(this.endpointI, body, options)
      .map(this.processHTTPResponseClient)
      .catch(this.processHTTPErrorClient);
    return obser;
  }
  postRequestII(creq: Request): Observable<Item> {
    let body = JSON.stringify(creq);
    let headers = new Headers({'Content-Type' : 'application/json'});
    let options = new RequestOptions({headers : headers});

    // this returns without waiting for POST response
    let obser = this.http.post(this.endpointII, body, options)
      .map(this.processHTTPResponseClient)
      .catch(this.processHTTPErrorClient);
    return obser;
  }

  processRequestI() {
    let val = this.inputValueI;
    let creq = new Request('', val);
    let srespFromSubscribe: Item;
    this.postRequestI(creq).subscribe(
      sresp => srespFromSubscribe = sresp,
      error => this.srespError = <any>error,
      () => this.processItemI(srespFromSubscribe), // on-complete
    );
    this.snackBar.open('Predicting correct words...', 'Requested!', {
      duration: 5000,
    });
    this.inProgressI = true;
    this.spinnerModeI = 'indeterminate';
  }
  processRequestII() {
    let val = this.inputValueII;
    let creq = new Request('', val);
    let srespFromSubscribe: Item;
    this.postRequestII(creq).subscribe(
      sresp => srespFromSubscribe = sresp,
      error => this.srespError = <any>error,
      () => this.processItemII(srespFromSubscribe), // on-complete
    );
    this.snackBar.open('Predicting next words...', 'Requested!', {
      duration: 5000,
    });
    this.inProgressII = true;
    this.spinnerModeII = 'indeterminate';
  }
}
