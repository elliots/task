/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
import * as vscode from 'vscode';
import { TaskTaskProvider } from './taskProvider';
import { CustomBuildTaskProvider } from './customTaskProvider';

let taskTaskProvider: vscode.Disposable | undefined;
let customTaskProvider: vscode.Disposable | undefined;

export function activate(_context: vscode.ExtensionContext): void {
	let workspaceRoot = vscode.workspace.rootPath;
	if (!workspaceRoot) {
		return;
	}
		
	taskTaskProvider = vscode.tasks.registerTaskProvider(TaskTaskProvider.TaskType, new TaskTaskProvider(workspaceRoot));
	customTaskProvider = vscode.tasks.registerTaskProvider(CustomBuildTaskProvider.CustomBuildScriptType, new CustomBuildTaskProvider(workspaceRoot));
}

export function deactivate(): void {
	if (taskTaskProvider) {
		taskTaskProvider.dispose();
	}
	if (customTaskProvider) {
		customTaskProvider.dispose();
	}
}