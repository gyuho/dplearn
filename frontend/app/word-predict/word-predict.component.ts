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

export class WordPredictRequest {
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

export class WordPredictResponse {
  result: string;
}

@Component({
  selector: 'app-word-predict',
  templateUrl: 'word-predict.component.html',
  styleUrls: ['word-predict.component.css'],
})
export class WordPredictComponent {
  mode = 'Observable';
  private wordPredictRequestEndpoint = 'word-predict-request';

  inputValueI: string;
  inputValueII: string;

  wordPredictResponse: WordPredictResponse;
  wordPredictResponseError: string;

  wordPredictResultI: string;
  wordPredictResultII: string;

  wordPredictInProgressI = false;
  spinnerColorI = 'primary';
  spinnerModeI = 'determinate';
  spinnerValueI = 0;

  wordPredictInProgressII = false;
  spinnerColorII = 'primary';
  spinnerModeII = 'determinate';
  spinnerValueII = 0;

  constructor(private http: Http, public snackBar: MdSnackBar) {
    this.inputValueI = '';
    this.inputValueII = '';
    this.wordPredictResponseError = '';
    this.wordPredictResultI = 'No results to show yet...';
    this.wordPredictResultII = 'No results to show yet...';
  }

  // ngOnInit(): void {}
  // ngAfterContentInit() {}
  // ngAfterViewInit() {}
  // ngAfterViewChecked() {}
  // ngOnDestroy() {
  //   console.log('Disconnected from cluster (user left the page)!');
  //   return;
  // }

  processWordPredictResponseI(resp: WordPredictResponse) {
    this.wordPredictResponse = resp;
    this.wordPredictResultI = resp.result;
    this.wordPredictInProgressI = false;
  }
  processWordPredictResponseII(resp: WordPredictResponse) {
    this.wordPredictResponse = resp;
    this.wordPredictResultII = resp.result;
    this.wordPredictInProgressII = false;
  }

  processHTTPResponseClient(res: Response) {
    let jsonBody = res.json();
    let wordPredictResponse = <WordPredictResponse>jsonBody;
    return wordPredictResponse || {};
  }

  processHTTPErrorClient(error: any) {
    let errMsg = (error.message) ? error.message :
      error.status ? `${error.status} - ${error.statusText}` : 'Server error';
    console.error(errMsg);
    this.wordPredictResponseError = errMsg;
    return Observable.throw(errMsg);
  }

  postRequest(wordPredictRequest: WordPredictRequest): Observable<WordPredictResponse> {
    let body = JSON.stringify(wordPredictRequest);
    let headers = new Headers({'Content-Type' : 'application/json'});
    let options = new RequestOptions({headers : headers});

    // this returns without waiting for POST response
    let obser = this.http.post(this.wordPredictRequestEndpoint, body, options)
      .map(this.processHTTPResponseClient)
      .catch(this.processHTTPErrorClient);
    return obser;
  }

  processRequestI() {
    let val = this.inputValueI;
    let wordPredictRequest = new WordPredictRequest(1, val);
    let wordPredictResponseFromSubscribe: WordPredictResponse;
    this.postRequest(wordPredictRequest).subscribe(
      wordPredictResponse => wordPredictResponseFromSubscribe = wordPredictResponse,
      error => this.wordPredictResponseError = <any>error,
      () => this.processWordPredictResponseI(wordPredictResponseFromSubscribe), // on-complete
    );
    this.snackBar.open('Predicting correct words...', 'Requested!', {
      duration: 2000,
    });
    this.wordPredictInProgressI = true;
    this.spinnerModeI = 'indeterminate';
  }
  processRequestII() {
    let val = this.inputValueII;
    let wordPredictRequest = new WordPredictRequest(2, val);
    let wordPredictResponseFromSubscribe: WordPredictResponse;
    this.postRequest(wordPredictRequest).subscribe(
      wordPredictResponse => wordPredictResponseFromSubscribe = wordPredictResponse,
      error => this.wordPredictResponseError = <any>error,
      () => this.processWordPredictResponseII(wordPredictResponseFromSubscribe), // on-complete
    );
    this.snackBar.open('Predicting next words...', 'Requested!', {
      duration: 2000,
    });
    this.wordPredictInProgressII = true;
    this.spinnerModeII = 'indeterminate';
  }
}
