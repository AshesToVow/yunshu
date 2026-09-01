// @ts-nocheck
import { createLazyLegacyPage, listLegacyPageIds } from './legacy-page-registry';
import type { ComponentType, LazyExoticComponent } from 'react';

export function listMenuPageComponentIds(): string[] {
  return listLegacyPageIds();
}

export function createLazyMenuPage(componentField: string): LazyExoticComponent<ComponentType<object>> | null {
  return createLazyLegacyPage(componentField);
}
