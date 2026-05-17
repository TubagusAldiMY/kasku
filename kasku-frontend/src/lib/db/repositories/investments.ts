import { createCrudRepository } from './crud';
import type { InvestmentRow } from '../schema';

export const investmentsRepo = createCrudRepository<InvestmentRow>('investments');
