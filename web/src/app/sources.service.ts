import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class SourcesService {
  constructor(private httpClient: HttpClient) {}

  getSources() : Observable<Source[]> {
    return this.httpClient.get<Source[]>('/sources')
  }

  addSource(source: Source) {
    return this.httpClient.post('/sources', source)
  }

  deleteSource(source: Source) {
    return this.httpClient.delete('/sources/' + source.id)
  }
}

export class Source {
  id!: number;
  name!: string;
  uri!: string;
  updateInterval!: string;
  lastUpdated!: string;
}
