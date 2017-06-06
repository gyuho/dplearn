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

export class CatsVsDogsRequest {
  url: string;
  rawdata: string;
  constructor(
    url: string,
    d: string,
  ) {
    this.url = url;
    this.rawdata = d;
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
export class CatsVsDogsComponent {
  mode = 'Observable';
  private catsVsDogsRequestEndpoint = 'cats-vs-dogs-request';

  inputValue: string;

  catsVsDogsResponse: CatsVsDogsResponse;
  catsVsDogsResponseError: string;

  catsVsDogsResult: string;
  catsVsDogsResultI: string;

  catsVsDogsInProgress = false;
  spinnerColor = 'primary';
  spinnerMode = 'determinate';
  spinnerValue = 0;

  constructor(private http: Http, public snackBar: MdSnackBar) {
    this.inputValue = '';
    this.catsVsDogsResponseError = '';
    this.catsVsDogsResult = 'No results to show yet...';
  }

  processCatsVsDogsResponse(resp: CatsVsDogsResponse) {
    this.catsVsDogsResponse = resp;
    this.catsVsDogsResult = resp.result;
    this.catsVsDogsInProgress = false;
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

  processRequest() {
    let val = this.inputValue;
    let catsVsDogsRequest = new CatsVsDogsRequest('http://aaa.com', val);
    let catsVsDogsResponseFromSubscribe: CatsVsDogsResponse;
    this.postRequest(catsVsDogsRequest).subscribe(
      catsVsDogsResponse => catsVsDogsResponseFromSubscribe = catsVsDogsResponse,
      error => this.catsVsDogsResponseError = <any>error,
      () => this.processCatsVsDogsResponse(catsVsDogsResponseFromSubscribe), // on-complete
    );
    this.snackBar.open('Predicting correct words...', 'Requested!', {
      duration: 5000,
    });
    this.catsVsDogsInProgress = true;
    this.spinnerMode = 'indeterminate';
  }
}
