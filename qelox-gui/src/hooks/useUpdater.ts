import { useEffect } from 'react';
import { check } from '@tauri-apps/plugin-updater';
import { useOrchestratorStore } from '../store';

export function useUpdater() {
    useEffect(() => {
        if (typeof window === 'undefined' || !('__TAURI_INTERNALS__' in window)) {
            return;
        }

        async function checkUpdate() {
            try {
                const update = await check();
                if (update) {
                    useOrchestratorStore.setState({
                        update: {
                            available: true,
                            version: update.version,
                            dismissed: false,
                        }
                    });
                }
            } catch (err) {
                console.error('Failed to check for updates:', err);
            }
        }

        checkUpdate();
        const interval = setInterval(checkUpdate, 1000 * 60 * 60); // Check every hour
        return () => clearInterval(interval);
    }, []);
}
