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

export class MNISTRequest {
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

export class MNISTResponse {
  result: string;
}

@Component({
  selector: 'app-mnist',
  templateUrl: 'mnist.component.html',
  styleUrls: ['mnist.component.css'],
})
export class MNISTComponent {
  mode = 'Observable';
  private mnistRequestEndpoint = 'mnist-request';

  inputValue: string;

  mnistResponse: MNISTResponse;
  mnistResponseError: string;

  mnistResult: string;

  mnistInProgress = false;
  spinnerColor = 'primary';
  spinnerMode = 'determinate';
  spinnerValue = 0;

  constructor(private http: Http, public snackBar: MdSnackBar) {
    this.inputValue = '';
    this.mnistResponseError = '';
    this.mnistResult = 'No results to show yet...';
  }

  processMNISTResponse(resp: MNISTResponse) {
    this.mnistResponse = resp;
    this.mnistResult = resp.result;
    this.mnistInProgress = false;
  }

  processHTTPResponseClient(res: Response) {
    let jsonBody = res.json();
    let mnistResponse = <MNISTResponse>jsonBody;
    return mnistResponse || {};
  }

  processHTTPErrorClient(error: any) {
    let errMsg = (error.message) ? error.message :
      error.status ? `${error.status} - ${error.statusText}` : 'Server error';
    console.error(errMsg);
    this.mnistResponseError = errMsg;
    return Observable.throw(errMsg);
  }

  postRequest(mnistRequest: MNISTRequest): Observable<MNISTResponse> {
    let body = JSON.stringify(mnistRequest);
    let headers = new Headers({'Content-Type' : 'application/json'});
    let options = new RequestOptions({headers : headers});

    // this returns without waiting for POST response
    let obser = this.http.post(this.mnistRequestEndpoint, body, options)
      .map(this.processHTTPResponseClient)
      .catch(this.processHTTPErrorClient);
    return obser;
  }

  processRequest() {
    let val = this.inputValue;
    let mnistRequest = new MNISTRequest(1, val);
    let mnistResponseFromSubscribe: MNISTResponse;
    this.postRequest(mnistRequest).subscribe(
      mnistResponse => mnistResponseFromSubscribe = mnistResponse,
      error => this.mnistResponseError = <any>error,
      () => this.processMNISTResponse(mnistResponseFromSubscribe), // on-complete
    );
    this.snackBar.open('Predicting correct words...', 'Requested!', {
      duration: 2000,
    });
    this.mnistInProgress = true;
    this.spinnerMode = 'indeterminate';
  }
}
