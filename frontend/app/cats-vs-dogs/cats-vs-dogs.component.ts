import {
  Component,
  OnInit,
  AfterContentInit,
  AfterViewChecked,
  AfterViewInit,
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

export class CatsVsDogsRequest {
  type: number;
  text: string;
  constructor(
    tp: number,
    txt: string,
  ) {
    this.type = tp;
    this.text = txt;
  }
}

export class CatsVsDogsResponse {
  result: string;
}

@Component({
  selector: 'app-cats-vs-dogs',
  templateUrl: 'cats-vs-dogs.component.html',
  styleUrls: ['cats-vs-dogs.component.css'],
})
export class CatsVsDogsComponent implements OnInit, AfterContentInit, AfterViewInit, AfterViewChecked, OnDestroy {
  mode = 'Observable';
  private catsVsDogsRequestEndpoint = 'cats-vs-dogs-request';

  inputValueI: string;
  inputValueII: string;

  catsVsDogsResponse: CatsVsDogsResponse;
  catsVsDogsResponseError: string;

  catsVsDogsResultI: string;
  catsVsDogsResultII: string;

  catsVsDogsIInProgress = false;
  spinnerColorI = 'primary';
  spinnerModeI = 'determinate';
  spinnerValueI = 0;

  catsVsDogsIIInProgress = false;
  spinnerColorII = 'primary';
  spinnerModeII = 'determinate';
  spinnerValueII = 0;

  constructor(private http: Http, public snackBar: MdSnackBar) {
    this.inputValueI = '';
    this.inputValueII = '';
    this.catsVsDogsResponseError = '';
    this.catsVsDogsResultI = 'Nothing to show...';
    this.catsVsDogsResultII = 'Nothing to show...';
  }

  ngOnInit(): void {}
  ngAfterContentInit() {}
  ngAfterViewInit() {}
  ngAfterViewChecked() {}

  // user leaves the template
  ngOnDestroy() {
    console.log('Disconnected from cluster (user left the page)!');
    return;
  }

  processCatsVsDogsResponseI(resp: CatsVsDogsResponse) {
    this.catsVsDogsResponse = resp;
    this.catsVsDogsResultI = resp.result;
    this.catsVsDogsIInProgress = false;
  }
  processCatsVsDogsResponseII(resp: CatsVsDogsResponse) {
    this.catsVsDogsResponse = resp;
    this.catsVsDogsResultII = resp.result;
    this.catsVsDogsIIInProgress = false;
  }

  processHTTPResponseClient(res: Response) {
    let jsonBody = res.json();
    let catsVsDogsResponse = <CatsVsDogsResponse>jsonBody;
    return catsVsDogsResponse || {};
  }

  processHTTPErrorClient(error: any) {
    let errMsg = (error.message) ? error.message :
      error.status ? `${error.status} - ${error.statusText}` : 'Server error';
    console.error(errMsg);
    this.catsVsDogsResponseError = errMsg;
    return Observable.throw(errMsg);
  }

  postRequest(catsVsDogsRequest: CatsVsDogsRequest): Observable<CatsVsDogsResponse> {
    let body = JSON.stringify(catsVsDogsRequest);
    let headers = new Headers({'Content-Type' : 'application/json'});
    let options = new RequestOptions({headers : headers});

    // this returns without waiting for POST response
    let obser = this.http.post(this.catsVsDogsRequestEndpoint, body, options)
      .map(this.processHTTPResponseClient)
      .catch(this.processHTTPErrorClient);
    return obser;
  }

  processRequestI() {
    let val = this.inputValueI;
    let catsVsDogsRequest = new CatsVsDogsRequest(1, val);
    let catsVsDogsResponseFromSubscribe: CatsVsDogsResponse;
    this.postRequest(catsVsDogsRequest).subscribe(
      catsVsDogsResponse => catsVsDogsResponseFromSubscribe = catsVsDogsResponse,
      error => this.catsVsDogsResponseError = <any>error,
      () => this.processCatsVsDogsResponseI(catsVsDogsResponseFromSubscribe), // on-complete
    );
    this.snackBar.open('Predicting correct words...', 'Requested!', {
      duration: 2000,
    });
    this.catsVsDogsIInProgress = true;
    this.spinnerModeI = 'indeterminate';
  }
  processRequestII() {
    let val = this.inputValueII;
    let catsVsDogsRequest = new CatsVsDogsRequest(2, val);
    let catsVsDogsResponseFromSubscribe: CatsVsDogsResponse;
    this.postRequest(catsVsDogsRequest).subscribe(
      catsVsDogsResponse => catsVsDogsResponseFromSubscribe = catsVsDogsResponse,
      error => this.catsVsDogsResponseError = <any>error,
      () => this.processCatsVsDogsResponseII(catsVsDogsResponseFromSubscribe), // on-complete
    );
    this.snackBar.open('Predicting next words...', 'Requested!', {
      duration: 2000,
    });
    this.catsVsDogsIIInProgress = true;
    this.spinnerModeII = 'indeterminate';
  }
}
