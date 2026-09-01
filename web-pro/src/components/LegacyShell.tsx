import type { PropsWithChildren } from 'react';
import { WorkloadProgressFloat } from '@/components/workload-progress-float';
import { AuthProvider } from '@/contexts/auth-context';
import { PluginProvider } from '@/contexts/plugin-context';
import { WorkloadProgressProvider } from '@/contexts/workload-progress-context';

/** 包裹旧版业务页所需的 Context（插件、发布进度、Auth 桥接） */
export function LegacyShell({ children }: PropsWithChildren) {
  return (
    <AuthProvider>
      <PluginProvider>
        <WorkloadProgressProvider>
          {children}
          <WorkloadProgressFloat />
        </WorkloadProgressProvider>
      </PluginProvider>
    </AuthProvider>
  );
}
