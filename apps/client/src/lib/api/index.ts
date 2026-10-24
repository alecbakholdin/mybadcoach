/**
 * Public surface of the client data layer: the service clients plus the error
 * helpers screens share. Import from `#lib/api` rather than reaching into the
 * individual modules.
 */
export * from './clients';
export * from './errors';
export * from './transport';
