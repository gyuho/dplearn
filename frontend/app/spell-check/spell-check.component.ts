import {
  Component,
  Injectable,
  OnInit,
  AfterContentInit,
  AfterViewChecked,
  ElementRef,
  ViewChild,
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

export class SpellCheckRequest {
  text: string;
  constructor(
    txt: string,
  ) {
    this.text = txt;
  }
}

export class SpellCheckResponse {
  text: string;
  result: string;
}

@Component({
  selector: 'app-spell-check',
  templateUrl: 'spell-check.component.html',
  styleUrls: ['spell-check.component.css'],
})
export class SpellCheckComponent implements OnInit, AfterContentInit, AfterViewChecked, OnDestroy {
  mode = 'Observable';
  private spellCheckRequestEndpoint = 'client-request';

  inputValue: string;

  spellCheckResponse: SpellCheckResponse;
  spellCheckResponseError: string;
  spellCheckResult: string;

  constructor(private http: Http) {
    this.inputValue = '';
    this.spellCheckResponseError = '';
    this.spellCheckResult = 'Nothing to show...';
  }

  ngOnInit(): void {}
  ngAfterContentInit() {}
  ngAfterViewChecked() {}

  // user leaves the template
  ngOnDestroy() {
    console.log('Disconnected from cluster (user left the page)!');
    return;
  }

  processSpellCheckResponse(resp: SpellCheckResponse) {
    this.spellCheckResponse = resp;
    this.spellCheckResult = resp.result;
  }

  processHTTPResponseClient(res: Response) {
    let jsonBody = res.json();
    let spellCheckResponse = <SpellCheckResponse>jsonBody;
    return spellCheckResponse || {};
  }

  processHTTPErrorClient(error: any) {
    let errMsg = (error.message) ? error.message :
      error.status ? `${error.status} - ${error.statusText}` : 'Server error';
    console.error(errMsg);
    this.spellCheckResponseError = errMsg;
    return Observable.throw(errMsg);
  }

  postRequest(spellCheckRequest: SpellCheckRequest): Observable<SpellCheckResponse> {
    let body = JSON.stringify(spellCheckRequest);
    let headers = new Headers({'Content-Type':'application/json'});
    let options = new RequestOptions({headers:headers});

    // this.spellCheckResult = 'Requested "' + spellCheckRequest.text + '"';

    // this returns without waiting for POST response
    let obser = this.http.post(this.spellCheckRequestEndpoint, body, options)
      .map(this.processHTTPResponseClient)
      .catch(this.processHTTPErrorClient);
    return obser;
  }

  // TODO: limit the text size
  processRequest() {
    let val = this.inputValue;
    let spellCheckRequest = new SpellCheckRequest(val);
    let spellCheckResponseFromSubscribe: SpellCheckResponse;
    this.postRequest(spellCheckRequest).subscribe(
      spellCheckResponse => spellCheckResponseFromSubscribe = spellCheckResponse,
      error => this.spellCheckResponseError = <any>error,
      () => this.processSpellCheckResponse(spellCheckResponseFromSubscribe), // on-complete
    );
  }
}
