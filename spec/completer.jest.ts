// import { beforeEach, describe, expect, it } from './lib/common';
import TemplateSrvStub from './lib/TemplateSrvStub';
import {Datasource} from '../src/module';
import {ConsulCompleter} from "../src/completer";
import q from 'q';
import {beforeEach, describe, expect, it} from "./lib/common";

describe('Completer', () => {
    const ctx: any = {
        backendSrv: {},
        templateSrv: new TemplateSrvStub(),
    };

    beforeEach(() => {
        ctx.$q = q;
        ctx.ds = new Datasource({}, ctx.$q, ctx.backendSrv, ctx.templateSrv);
        ctx.completer = new ConsulCompleter(ctx.ds, ctx.templateSrv);
    });

    it('getCompletions should return completions', (done) => {
        ctx.backendSrv.datasourceRequest = function (request) {
            expect(request.data.queries[0].target).toBe("registry/");
            return ctx.$q.when({
                _request: request,
                data: {
                    results: {
                        "keys": {
                            refId: 'keys',
                            series:
                                [
                                    {
                                        name: 'registry/v1',
                                        points: [1]
                                    },
                                    {
                                        name: 'registry/v2',
                                        points: [1]
                                    },
                                    {
                                        name: 'registry/v3',
                                        points: [1]
                                    }
                                ]
                        }
                    }
                }
            });
        };

        ctx.templateSrv.replace = function (data) {
            return data;
        };

        ctx.completer.getCompletions(null,
            {
                getTokenAt: (row, col) => {
                    expect(row).toBe(10);
                    expect(col).toBe(11);
                    return {value: "registry/"}
                }
            },
            {row: 10, column: 11}, "registry/",
            (err, completions) => {
                expect(err).toBeNull();
                expect(completions[0].caption).toBe("registry/v1");
                expect(completions[0].meta).toBe("key");
                expect(completions[0].value).toBe("v1");

                expect(completions[1].caption).toBe("registry/v2");
                expect(completions[1].meta).toBe("key");
                expect(completions[1].value).toBe("v2");

                expect(completions[2].caption).toBe("registry/v3");
                expect(completions[2].meta).toBe("key");
                expect(completions[2].value).toBe("v3");
                done()
            }
        )
    });

});
