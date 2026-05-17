import { createCrudRepository } from './crud';
import type { AccountRow } from '../schema';

export const accountsRepo = createCrudRepository<AccountRow>('accounts');
