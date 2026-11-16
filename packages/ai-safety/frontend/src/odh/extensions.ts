import type {
  NavExtension,
  RouteExtension,
  AreaExtension,
} from '@odh-dashboard/plugin-core/extension-points';
// The SupportedArea entry and feature flag must be registered in the host before building:
// see the odh-dashboard /module-onboarding skill (Phase 3).
import { SupportedArea } from '@odh-dashboard/plugin-core/areas';

const extensions: (NavExtension | RouteExtension | AreaExtension)[] = [
  {
    type: 'app.area',
    properties: {
      id: SupportedArea.PLUGIN_AI_SAFETY,
      featureFlags: ['aiSafety'],
    },
  },
  {
    type: 'app.navigation/section',
    flags: {
      required: [SupportedArea.PLUGIN_AI_SAFETY],
    },
    properties: {
      id: 'ai-safety',
      title: 'AI Safety Studio',
      group: '7_ai_safety_studio',
      iconRef: () => import('./AiSafetyNavIcon'),
    },
  },
  {
    type: 'app.navigation/href',
    flags: {
      required: [SupportedArea.PLUGIN_AI_SAFETY],
    },
    properties: {
      id: 'ai-safety-guardrails-view',
      title: 'Guardrails',
      href: '/ai-safety/guardrails',
      section: 'ai-safety',
      path: '/ai-safety/guardrails/*',
      label: 'Tech Preview',
    },
  },
  {
    type: 'app.route',
    flags: {
      required: [SupportedArea.PLUGIN_AI_SAFETY],
    },
    properties: {
      path: '/ai-safety/guardrails/*',
      component: () => import('./AiSafetyGuardrailsWrapper'),
    },
  },
  {
    type: 'app.route',
    flags: {
      required: [SupportedArea.PLUGIN_AI_SAFETY],
    },
    properties: {
      path: '/ai-safety/main-view/*',
      component: () => import('./AiSafetyWrapper'),
    },
  },
];

export default extensions;
