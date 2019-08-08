import * as path from 'path';
import * as fs from 'fs';
import * as cp from 'child_process';
import * as vscode from 'vscode';

export class TaskTaskProvider implements vscode.TaskProvider {
	static TaskType: string = 'task';
	private taskPromise: Thenable<vscode.Task[]> | undefined = undefined;

	constructor(workspaceRoot: string) {
		let pattern = path.join(workspaceRoot, 'Taskfile.yml');
		let fileWatcher = vscode.workspace.createFileSystemWatcher(pattern);
		fileWatcher.onDidChange(() => this.taskPromise = undefined);
		fileWatcher.onDidCreate(() => this.taskPromise = undefined);
		fileWatcher.onDidDelete(() => this.taskPromise = undefined);
	}

	public provideTasks(): Thenable<vscode.Task[]> | undefined {
		if (!this.taskPromise) {
			this.taskPromise = getTaskTasks();
		}
		return this.taskPromise;
	}

	public resolveTask(_task: vscode.Task): vscode.Task | undefined {
		const task = _task.definition.task;
		// A Task task consists of a task and an optional file as specified in TaskTaskDefinition
		// Make sure that this looks like a Task task by checking that there is a task.
		if (task) {
			// resolveTask requires that the same definition object be used.
			const definition: TaskTaskDefinition = <any>_task.definition;
			return new vscode.Task(definition, definition.task, 'task', new vscode.ShellExecution(`task ${definition.task}`));
		}
		return undefined;
	}
}

function exists(file: string): Promise<boolean> {
	return new Promise<boolean>((resolve, _reject) => {
		fs.exists(file, (value) => {
			resolve(value);
		});
	});
}

function exec(command: string, options: cp.ExecOptions): Promise<{ stdout: string; stderr: string }> {
	return new Promise<{ stdout: string; stderr: string }>((resolve, reject) => {
		cp.exec(command, options, (error, stdout, stderr) => {
			if (error) {
				reject({ error, stdout, stderr });
			}
			resolve({ stdout, stderr });
		});
	});
}

let _channel: vscode.OutputChannel;
function getOutputChannel(): vscode.OutputChannel {
	if (!_channel) {
		_channel = vscode.window.createOutputChannel('Task Auto Detection');
	}
	return _channel;
}

interface TaskTaskDefinition extends vscode.TaskDefinition {
	/**
	 * The task name
	 */
	task: string;

	/**
	 * The description of the task
	 */
	description?: string;
}

const buildNames: string[] = ['build', 'compile', 'watch'];
function isBuildTask(name: string): boolean {
	for (let buildName of buildNames) {
		if (name.indexOf(buildName) !== -1) {
			return true;
		}
	}
	return false;
}

const testNames: string[] = ['test', 'lint'];
function isTestTask(name: string): boolean {
	for (let testName of testNames) {
		if (name.indexOf(testName) !== -1) {
			return true;
		}
	}
	return false;
}

function getProblemMatchers(t: any): Array<string> {
	// TODO:
	return [];
}

async function getTaskTasks(): Promise<vscode.Task[]> {
	let workspaceRoot = vscode.workspace.rootPath;
	let emptyTasks: vscode.Task[] = [];
	if (!workspaceRoot) {
		return emptyTasks;
	}
	
	let commandLine = 'task --json';
	getOutputChannel().appendLine('running command');
	try {
		let { stdout, stderr } = await exec(commandLine, { cwd: workspaceRoot });
		if (stderr && stderr.length > 0) {
			getOutputChannel().appendLine(stderr);
			getOutputChannel().show(true);
		}
		let result: vscode.Task[] = [];
		if (stdout) {
			//let channel = getOutputChannel();
			// channel.appendLine('Got tasks as json');
			// channel.appendLine(stdout);
			// channel.show(true);

			var taskResults = JSON.parse(stdout);

			result = taskResults.map((t:any) => {
				let kind: TaskTaskDefinition = {
					type: 'task',
					task: t.Task,
					description: t.Desc,
				};
				let task = new vscode.Task(kind, t.Task, 'task', new vscode.ShellExecution(`task ${t.Task}`), getProblemMatchers(t));

				let lowerCaseLine = t.Task.toLowerCase();
				if (isBuildTask(lowerCaseLine)) {
					task.group = vscode.TaskGroup.Build;
				} else if (isTestTask(lowerCaseLine)) {
					task.group = vscode.TaskGroup.Test;
				}

				return task;
			})

			// let lines = stdout.split(/\r{0,1}\n/);
			// for (let line of lines) {
			// 	if (line.length === 0) {
			// 		continue; 
			// 	}
			// 	let regExp = /task\s(.*)#/;
			// 	let matches = regExp.exec(line);
			// 	if (matches && matches.length === 2) {
			// 		let taskName = matches[1].trim();
			// 		let kind: TaskTaskDefinition = {
			// 			type: 'task',
			// 			task: taskName
			// 		};
			// 		let task = new vscode.Task(kind, taskName, 'task', new vscode.ShellExecution(`task ${taskName}`));
			// 		result.push(task);
			// 		let lowerCaseLine = line.toLowerCase();
			// 		if (isBuildTask(lowerCaseLine)) {
			// 			task.group = vscode.TaskGroup.Build;
			// 		} else if (isTestTask(lowerCaseLine)) {
			// 			task.group = vscode.TaskGroup.Test;
			// 		}
			// 	}
			// }
		}
		return result;
	} catch (err) {
		let channel = getOutputChannel();
		if (err.stderr) {
			channel.appendLine(err.stderr);
		}
		if (err.stdout) {
			channel.appendLine(err.stdout);
		}
		channel.appendLine('Auto detecting task tasts failed.');
		channel.show(true);
		return emptyTasks;
	}
}
